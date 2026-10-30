package state

import (
	"fmt"
	"strings"
)

const retiredCoursesKey = "retired_courses"

func SaveCourse(slug, name, body string) error {
	dbPath, err := openDB()
	if err != nil {
		return err
	}
	return upsertCourse(dbPath, slug, name, body, 0, true)
}

func CreateCourse(name, body string) (string, error) {
	dbPath, err := openDB()
	if err != nil {
		return "", err
	}
	slug := slugFor(name)
	_, ok, err := courseBySlug(dbPath, slug)
	if err != nil {
		return "", err
	}
	if ok {
		return "", fmt.Errorf("course already exists")
	}
	if err := upsertCourse(dbPath, slug, name, body, 0, true); err != nil {
		return "", err
	}
	return slug, nil
}

func DeleteStoredCourse(slug string) error {
	dbPath, err := openDB()
	if err != nil {
		return err
	}
	embedded, err := readEmbedded()
	if err != nil {
		return err
	}
	for _, item := range embedded {
		if item.Slug == slug {
			if err := retireSlug(dbPath, slug); err != nil {
				return err
			}
			break
		}
	}
	return execSQL(dbPath, "DELETE FROM course WHERE slug = "+sqlString(slug)+";\n")
}

func slugRetired(dbPath, slug string) (bool, error) {
	value, err := metaValue(dbPath, retiredCoursesKey)
	if err != nil {
		return false, err
	}
	for _, item := range strings.Split(value, "\n") {
		if item == slug {
			return true, nil
		}
	}
	return false, nil
}

func retireSlug(dbPath, slug string) error {
	retired, err := slugRetired(dbPath, slug)
	if err != nil {
		return err
	}
	if retired {
		return nil
	}
	value, err := metaValue(dbPath, retiredCoursesKey)
	if err != nil {
		return err
	}
	if strings.TrimSpace(value) == "" {
		value = slug
	} else {
		value = value + "\n" + slug
	}
	return setMeta(dbPath, retiredCoursesKey, value)
}
