<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#current-function?>
    <!-- Purpose: Test for a variable reference in the middle of a path expression. -->

  <xsl:variable name="data">
    <bananas/>
  </xsl:variable>
  
  <xsl:template match="doc">
  <out>
    <yes><xsl:copy-of select="foo/item[@type='z']/$data"/></yes>
    <no><xsl:copy-of select="foo/item[@type='j']/$data"/></no>
  </out>
</xsl:template>

</xsl:stylesheet>
