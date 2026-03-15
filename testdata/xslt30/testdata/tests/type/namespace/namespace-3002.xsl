<xsl:stylesheet version="2.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<!-- test case for spec bug 5857: namespace undeclaration on a copied
     element is not copied through to the output -->
<!-- Author: Michael Kay -->
<!-- Date: 2 February 2009 -->

<xsl:template match="/">
  <xsl:apply-templates/>
</xsl:template>

<xsl:template match="*">
  <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
  </xsl:copy>
</xsl:template>

  
</xsl:stylesheet>
