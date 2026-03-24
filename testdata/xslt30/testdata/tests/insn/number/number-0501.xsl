<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:fo="http://www.w3.org/XSL/Format/1.0">

  <!-- From example in draft -->
  <xsl:template match="doc">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>
 
  <xsl:template match="H4">
   <fo:block>
     <xsl:number level="any" from="H1" count="H2"/>
     <xsl:text>.</xsl:text>
     <xsl:number level="any" from="H2" count="H3"/>
     <xsl:text>.</xsl:text>
     <xsl:number level="any" from="H3" count="H4"/>
     <xsl:text>.</xsl:text>
     <xsl:number level="any" from="H3" count="EMPTY"/>
     <xsl:text> </xsl:text>
     <xsl:apply-templates/>
   </fo:block>
  </xsl:template>

</xsl:stylesheet>
