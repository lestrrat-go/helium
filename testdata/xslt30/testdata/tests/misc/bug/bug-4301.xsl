<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" >

<?spec xpath#axes?>
<!-- BUG: 6.0/006 -->
<!-- DESCRIPTION: With TinyTree model, descendants axis gives wrong answer if there
     are no descendants or following siblings -->

<xsl:template match="/">
<doc>
 <xsl:apply-templates select="//icon"/>
</doc>
</xsl:template>


<xsl:template match="icon">
 <xsl:if test="count(.//text())>0"> 
  <message>icon <xsl:number level="any"/> has content </message>
 </xsl:if>
</xsl:template>
</xsl:stylesheet>
