<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
                xmlns:foo="http://foo.com">

  <!-- FileName: NUMBERFORMAT20 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.3 -->
    <!-- Purpose: Test of decimal-format with qualified name. Unqualified name provided as a trap. -->

<xsl:decimal-format name="foo:decimal1" decimal-separator='!' grouping-separator='*'/>
<xsl:decimal-format name="decimal1" decimal-separator='*' grouping-separator='!'/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(1234.567,'#*###*###!###','foo:decimal1')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
