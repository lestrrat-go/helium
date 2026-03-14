<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:fo="http://www.w3.org/XSL/Format/1.0">

  <!-- test xsl:number with a pattern containing a variable reference -->
  <xsl:template match="doc">
    <out>
      <xsl:apply-templates select="chapter/note"/>
    </out>
  </xsl:template>
  
  <xsl:param name="v1" select="'chapter'"/>
  <xsl:param name="v2" select="'doc'"/>
    
  <xsl:template match="note">
    <xsl:param name="v3" select="(position()-1) idiv 3 + 1"/>
    <fo:block>
       <xsl:number level="any" from="*[name()=$v1]" format="(1) "/>
       <xsl:number level="any" from="*[name()=$v2]" format="[1] "/>
       <xsl:number level="any" from="chapter[$v3]" format="{{1}} "/>       
       <xsl:apply-templates/>
    </fo:block>;
  </xsl:template>
  
  <xsl:strip-space elements="*"/>
 
</xsl:stylesheet>
