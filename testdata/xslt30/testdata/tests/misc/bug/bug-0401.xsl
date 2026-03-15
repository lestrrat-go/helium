<?xml version='1.0'?>
<xsl:stylesheet version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#patterns?>
<!-- BUG: Saxon 5.3/004 -->
<!-- DESCRIPTION: The priorities of the components in a union pattern may be calculated incorrectly,
    for example in the pattern "text() | *" both components are given priority +0.5 instead
    of -0.5. -->


<xsl:template match="nos" >
  <src>OK</src>
</xsl:template>

<xsl:template match="*|text()">
  <src>BUG</src>
</xsl:template>

</xsl:stylesheet>
