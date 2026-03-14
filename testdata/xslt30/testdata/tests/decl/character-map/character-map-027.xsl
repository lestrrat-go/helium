<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test of functions in predicate. -->

<xsl:template name="go">
  <xsl:character-map name="currency">
    <xsl:output-character character="$" string="£"/>
  </xsl:character-map>
  <out>
    <xsl:value-of select="."/>
  </out>
</xsl:template>


</xsl:stylesheet>
