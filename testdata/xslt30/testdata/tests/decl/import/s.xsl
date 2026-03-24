<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- FileName: s -->
<!-- Purpose: Used by impincl15. -->

<xsl:template match="tag" mode="yes">
  <pre><xsl:apply-templates/></pre>
</xsl:template>

<xsl:template match="tag" mode="no">
  <bad><xsl:apply-templates/></bad>
</xsl:template>

<xsl:template match="tag">
  <unmoded><xsl:apply-templates/></unmoded>
</xsl:template>

</xsl:stylesheet>