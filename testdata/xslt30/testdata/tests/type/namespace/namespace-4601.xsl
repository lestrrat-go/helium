<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:pre="http://example.com"
  exclude-result-prefixes="pre">

<?spec fo#func-namespace-uri?>
<?spec fo#func-local-name?>
<?spec fo#func-name?>
    <!-- Purpose: Test name functions on default-namespace declaration. Should be null strings. -->

<xsl:output method="xml" encoding="UTF-8" indent="no" />

<xsl:template match="pre:doc">
  <out>
    <xsl:value-of select="count(namespace::*[name(.)!='xml'])"/><xsl:text> namespace node qualifies:&#10;</xsl:text>
    <xsl:text>name|</xsl:text><xsl:value-of select="name(namespace::*[name(.)!='xml'])"/><xsl:text>|</xsl:text>
    <xsl:text>namespace-uri|</xsl:text><xsl:value-of select="namespace-uri(namespace::*[name(.)!='xml'])"/><xsl:text>|</xsl:text>
    <xsl:text>local-name|</xsl:text><xsl:value-of select="local-name(namespace::*[name(.)!='xml'])"/><xsl:text>|</xsl:text>
  </out>
</xsl:template>

</xsl:stylesheet>