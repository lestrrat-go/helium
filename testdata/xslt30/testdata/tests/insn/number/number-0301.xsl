<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:fo="http://www.w3.org/XSL/Format/1.0">

  <!-- PURPOSE -->
  <xsl:template match="doc">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>
  
  <xsl:template match="title">
    <fo:block>
       <xsl:number level="multiple"
                   count="chapter|section|subsection"
                   format="1.1. "/>
       <xsl:apply-templates/>
    </fo:block>
  </xsl:template>

  <xsl:template match="appendix//title" priority="1">
    <fo:block>
       <xsl:number level="multiple"
                   count="appendix|section|subsection"
                   format="A.1. "/>
       <xsl:apply-templates/>
    </fo:block>
  </xsl:template>
 
</xsl:stylesheet>
