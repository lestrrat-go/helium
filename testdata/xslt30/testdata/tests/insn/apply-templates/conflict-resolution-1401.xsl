<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
		xmlns="http://www.w3.org/1999/xhtml"
		xmlns:h="http://www.w3.org/1999/xhtml"
		xmlns:f="http://docbook.org/xslt/ns/extension"
		xmlns:m="http://docbook.org/xslt/ns/mode"
		xmlns:fn="http://www.w3.org/2003/11/xpath-functions"
		xmlns:db="http://docbook.org/docbook-ng"
		exclude-result-prefixes="h f m fn db"
                version="2.0">
                
                <?spec xslt#modes?>

<xsl:variable name="dummy">
  <db:book>
      <db:info>
	<db:title>Book Title</db:title>
      </db:info>
    <db:chapter>
      <db:info>
	<db:title>ChapterTitle</db:title>
      </db:info>
      <db:para/>
    </db:chapter>
  </db:book>
</xsl:variable>

<xsl:template match="/">
  <xsl:apply-templates select="$dummy/db:book/db:chapter/db:info/db:title"
		       mode="m:titlepage-mode"/>
</xsl:template>

<xsl:template match="db:chapter/db:info/db:title
		     |db:appendix/db:info/db:title
		     |db:preface/db:info/db:title
		     |db:bibliography/db:info/db:title"
	      mode="m:titlepage-mode"
	      priority="100">

  <h2>
    <xsl:next-match/>
  </h2>
</xsl:template>

<xsl:template match="db:title" mode="m:titlepage-mode">
  <xsl:apply-templates/>
</xsl:template>

<xsl:template match="*" mode="m:titlepage-mode">
  <xsl:apply-templates select="."/>
</xsl:template>

</xsl:stylesheet>
