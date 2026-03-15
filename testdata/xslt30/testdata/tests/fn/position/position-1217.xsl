<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position39 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
    <!-- Purpose: Test of shorthand positional indexing in 2nd predicate. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="*[@test][3]/num"/>
  </out>
</xsl:template>

</xsl:stylesheet>
