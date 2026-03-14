<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#generate-id?>
    <!-- Purpose: Test of generate-id() uniqueness. -->
  <!-- All IDs should be different. -->

<xsl:output method="xml" encoding="UTF-8" indent="no"/>

<xsl:template match="doc">
  <out>
    <xsl:choose>
      <xsl:when test="generate-id() = generate-id(a)">
        <xsl:text>FAIL on parent-child 'a' element: </xsl:text>
        <xsl:value-of select="generate-id()"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(a)"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id() = generate-id(b)">
        <xsl:text>FAIL on parent-child 'b' element: </xsl:text>
        <xsl:value-of select="generate-id()"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(b)"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id(a) = generate-id(b)">
        <xsl:text>FAIL on child 'a' to child 'b' element: </xsl:text>
        <xsl:value-of select="generate-id(a)"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(b)"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id() = generate-id(./@att)">
        <xsl:text>FAIL on parent-attribute: </xsl:text>
        <xsl:value-of select="generate-id()"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(./@att)"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id(./@att) = generate-id(a)">
        <xsl:text>FAIL on attribute-child 'a' element: </xsl:text>
        <xsl:value-of select="generate-id(./@att)"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(a)"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id(./@att) = generate-id(b)">
        <xsl:text>FAIL on attribute-child 'b' element: </xsl:text>
        <xsl:value-of select="generate-id(./@att)"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(b)"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id() = generate-id(./text()[1])">
        <xsl:text>FAIL on parent-text: </xsl:text>
        <xsl:value-of select="generate-id()"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(./text()[1])"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id(./text()[1]) = generate-id(a)">
        <xsl:text>FAIL on text-child 'a' element: </xsl:text>
        <xsl:value-of select="generate-id(./text()[1])"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(a)"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id(./text()[1]) = generate-id(b)">
        <xsl:text>FAIL on text-child 'b' element: </xsl:text>
        <xsl:value-of select="generate-id(./text()[1])"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(b)"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:when test="generate-id(./@att) = generate-id(./text()[1])">
        <xsl:text>FAIL on attribute-text: </xsl:text>
        <xsl:value-of select="generate-id(./@att)"/><xsl:text>,  </xsl:text>
        <xsl:value-of select="generate-id(./text()[1])"/><xsl:text>&#10;</xsl:text>
      </xsl:when>
      <xsl:otherwise>
        <xsl:text>Success</xsl:text>
      </xsl:otherwise>
    </xsl:choose>
  </out>
</xsl:template>

</xsl:stylesheet>