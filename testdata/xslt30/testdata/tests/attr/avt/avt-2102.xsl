<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:http="http://xyz.com/"
  exclude-result-prefixes="http">

<?spec xslt#attribute-value-templates?>
    <!-- Purpose: Absent expression in AVT (bug 29226). -->

<xsl:template match="doc">
  <out a="x{}y{}z" b="{ }" c="x{(:comment:)}y{  (:comment:) }z" d="{ (:comment:)(:comment:) (:comment(:oh!:):)}" e="{ (:}}:) }"/>
</xsl:template>

</xsl:stylesheet>