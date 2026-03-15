<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position75 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of last() on various sets of children. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="a/*[last()]"/>,
    <xsl:value-of select="a/child::*[last()]"/>,
    <xsl:value-of select="a/descendant::*[last()]"/>;
    <xsl:value-of select="a/child::node()[last()]"/>,
    <xsl:value-of select="a/child::text()[last()]"/>;
    <xsl:value-of select="a/child::comment()[last()]"/>,
  </out>
</xsl:template>

</xsl:stylesheet>
