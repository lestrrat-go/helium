<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:http="http://xyz.com/"
  exclude-result-prefixes="http">

<?spec xslt#attribute-value-templates?>
    <!-- Purpose: Use of colon in AVT expression to signify namespaced element. -->

<xsl:template match="doc">
  <out href="{http:val}"/>
</xsl:template>

</xsl:stylesheet>