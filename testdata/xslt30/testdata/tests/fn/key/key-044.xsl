<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Purpose: Test two calls to generate-id() on the same file. All IDs should be
    the same. If both filenames were given literally, the spec says that IDs must match.
    Retrieving the name from the principal XML document should still clearly mean the
    same file. -->

<xsl:output method="xml" encoding="UTF-8" indent="no"/>

<xsl:template match="doc">
  <out>
    <xsl:variable name="tn" select="document('key151a.xml')//(doc/body/text())[1]" />
    <xsl:choose>
      <xsl:when test="generate-id(document('key151a.xml')//body) != generate-id(document(a)//body)">
        <xsl:text>FAIL on body element: </xsl:text>
        <xsl:value-of select="generate-id(document('key151c.xml')//body)"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(document(a)//body)"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id((document('key151a.xml')//text())[1]) != generate-id((document(a)//text())[1])">
        <xsl:text>FAIL on first text node: </xsl:text>
        <xsl:value-of select="generate-id(document('key151c.xml')//text()[1])"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(document(a)//text()[1])"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id(document('key151a.xml')//body/@att) != generate-id(document(a)//body/@att)">
        <xsl:text>FAIL on body attribute: </xsl:text>
        <xsl:value-of select="generate-id(document('key151c.xml')//body/@att)"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(document(a)//body/@att)"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id((document('key151a.xml')//comment())[1]) != generate-id(document(a)//comment()[1])">
        <xsl:text>FAIL on first comment node: </xsl:text>
        <xsl:value-of select="generate-id((document('key151c.xml')//comment())[1])"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id((document(a)//comment())[1])"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <!-- Now try the node in the variable -->
      <xsl:when test="generate-id($tn) != generate-id((document(a)//doc/body/text())[1])">
        <xsl:text>FAIL on inner text node: </xsl:text>
        <xsl:value-of select="generate-id((document('key151c.xml')//text())[1])"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id((document(a)//text())[1])"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:otherwise>
        <xsl:text>Success</xsl:text>
      </xsl:otherwise>
    </xsl:choose>
  </out>
</xsl:template>

</xsl:stylesheet>