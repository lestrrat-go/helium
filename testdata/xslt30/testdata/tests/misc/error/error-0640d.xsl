<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test use of a variable within a pattern causing a circularity (XSLT 2.0) -->
<?spec xslt#errors?><?error XTDE0640?>

   <xsl:param name="p">
      <xsl:apply-templates select="//param"/>
   </xsl:param>

   <xsl:template match="*">
      <xsl:copy>
         <xsl:apply-templates/>
      </xsl:copy>
   </xsl:template>
  
   <xsl:template match="param[@id=$p]">2</xsl:template>
   <xsl:template match="param[@id!=$p]">3</xsl:template>

   <xsl:template match="param">
  <!-- Under the provisions of section 5.5.4, a processor can recover from the circularity errors in matching
       the above patterns by treating the rules as not matching. To make sure that this test always returns an
       error result, we therefore force a circularity in this fallback rule. -->
       
      <xsl:value-of select="count($p)"/>
   </xsl:template>     

</xsl:stylesheet>
