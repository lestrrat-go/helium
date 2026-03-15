<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-if?>
  <!-- Purpose: Test xsl:if with equality of result tree fragments. -->

<xsl:variable name="v1">test</xsl:variable>
<xsl:variable name="v2">test</xsl:variable>

<xsl:template match="/">
  <out>
    <xsl:if test="$v1=$v2">Success1</xsl:if>
    <xsl:if test="string($v1)=string($v2)">Success2</xsl:if>
  </out>
</xsl:template>

</xsl:stylesheet>
