<?xml version="1.0"?> 
<xsl:stylesheet 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  version="3.0">

  <!-- PURPOSE trigger Saxon bug 6003 - xsl:evaluate applied to an inline function expression, whose closure 
       includes one of the supplied xsl:evaluate parameters -->


  <xsl:param name="p">function($x){$x + $y}</xsl:param>
  
  <xsl:template name="xsl:initial-template">
    <out>
       <xsl:variable name="f" as="function(*)">
         <xsl:evaluate xpath="$p">
           <xsl:with-param name="y" select="42"/>
         </xsl:evaluate>
       </xsl:variable>
       <in><xsl:value-of select="$f(18)"/></in> 
    </out>
  </xsl:template>

 
</xsl:stylesheet>
