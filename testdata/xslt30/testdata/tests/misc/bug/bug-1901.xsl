<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns="http://www.w3.org/1999/xhtml">

<!-- BUG: Turned out not to be a bug -->

<!-- Create output document template -->
<xsl:template match="/">
  <html>
  <head>
    <title>test</title>
  </head>
  <body>
    <xsl:call-template name="generate-glossary"/>
  </body>
  </html>
</xsl:template>

<!-- Generate glossary -->
<xsl:template name="generate-glossary">
  <dl>
    <xsl:apply-templates select="//gi-def" mode="build-glossary"/>
  </dl>
</xsl:template>

<!-- Process one gi-def for glossary output -->
<xsl:template match="gi-def" mode="build-glossary">
  <dt>
    <xsl:apply-templates select="t" mode="build-glossary"/>
  </dt>
  <dd>
    <xsl:apply-templates select="def" mode="build-glossary"/>
  </dd>
</xsl:template>

<!-- Process term, this works just right -->
<xsl:template match="gi-def/t" mode="build-glossary">
  <xsl:value-of select=".//text()"/>
</xsl:template>

<!-- Process definition, THIS IS LOST!!! Huh??? -->
<xsl:template match="gi-def/def" mode="build-glossary">
  <div>
    <xsl:value-of select=".//text()"/>
  </div>
</xsl:template>

</xsl:stylesheet>
