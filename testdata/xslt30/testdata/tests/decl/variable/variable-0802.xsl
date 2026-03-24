<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#variable-values?>
  <!-- Purpose: Test for passing variable into other template via with-param.  -->
  <!-- Author: Paul Dick -->

<xsl:template match="doc">
   <out>
     <xsl:variable name="ResultTreeFragTest">
        <B><xsl:value-of select="a"/></B>
     </xsl:variable>
     <xsl:apply-templates select="a">
        <xsl:with-param name="pvar1" select="$ResultTreeFragTest"/>
     </xsl:apply-templates>
   </out>
</xsl:template>

<xsl:template match="a">
   <xsl:param name="pvar1">Default text in pvar1</xsl:param>
   <xsl:param name="pvar2">Default text in pvar2</xsl:param>
   <xsl:value-of select="$pvar1"/>, <xsl:value-of select="$pvar2"/>
</xsl:template>

</xsl:stylesheet>