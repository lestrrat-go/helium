<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:fo="http://www.w3.org/XSL/Format/1.0">

<!-- PURPOSE: test numeric predicates in match patterns -->

  <xsl:template match="doc">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

<xsl:template match="*[2]" priority="2">
1: <xsl:copy>
    <xsl:apply-templates/>
    </xsl:copy>
</xsl:template>

<xsl:template match="*">
n: <xsl:copy>
    <xsl:apply-templates/>
    </xsl:copy>
</xsl:template>


  <xsl:template match="H4XX">
   <fo:block>
   <xsl:variable name="n" select="1"/>
     <xsl:number level="any" from="H3[1]" count="H4"/>
     <xsl:apply-templates/>
   </fo:block>
  </xsl:template>

</xsl:stylesheet>
