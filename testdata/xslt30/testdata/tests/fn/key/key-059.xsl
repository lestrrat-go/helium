<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- tests the use of variables within xsl:key match pattern. -->
<?spec xslt#xsl-key?>
  <xsl:template match="text()" />

  <xsl:variable name="prefix">a:</xsl:variable>

  <xsl:key name="targets"
    match="target[starts-with(@ref, $prefix)]"
    use="1" />

  <xsl:template match="/">
  <out><xsl:value-of select="key('targets', 1)/@name"/></out>
  </xsl:template>

</xsl:stylesheet>
