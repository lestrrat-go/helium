<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test of functions in predicate. -->

<xsl:template match="letter[position()=last()]">
  <out>
    <xsl:value-of select="."/>
  </out>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>
