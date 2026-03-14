<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
<xsl:template match="/doc">
 <out>
   <xsl:for-each select="person">
     <xsl:choose>
           <xsl:when test="sex='M'">&#xa;Male: </xsl:when>
           <xsl:when test="sex='F'">&#xa;Female: </xsl:when>
       <xsl:otherwise>&#xa;Who knows?: </xsl:otherwise>
     </xsl:choose>
	 <xsl:value-of select="name"/>
   </xsl:for-each>
 </out>
</xsl:template>

</xsl:stylesheet>