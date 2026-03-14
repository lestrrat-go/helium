<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:output encoding="iso-8859-1"/>

  <!-- PURPOSE: test ordinal numbering in Italian words, using CLDR numbering scheme name -->
  
  <xsl:template match="doc">
    <out>
       <m>
           <xsl:for-each select="1 to 10">
             <xsl:number value="." format="w" lang="it" ordinal="%spellout-ordinal-masculine"/>
             <xsl:text>;</xsl:text>
           </xsl:for-each>
      </m>
      <f>
           <xsl:for-each select="1 to 10">
             <xsl:number value="." format="w" lang="it" ordinal="%spellout-ordinal-feminine"/>
             <xsl:text>;</xsl:text>
           </xsl:for-each>
      </f>
    </out>  
  </xsl:template>
 
</xsl:stylesheet>
