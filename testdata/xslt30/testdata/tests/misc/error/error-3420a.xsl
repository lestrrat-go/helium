<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!--If no enclosing node N of the [accumulator-after()] function call has a preceding sibling node P such that 
      (a) N and P are part of the same sequence constructor, and 
      (b) the sweep of P is consuming, 
      then the function call is consuming. 
  
      So the call of accumulator-after() is consuming, 
      and since xsl:copy-of is also consuming, the construct fails
      the streamability rules because there are two consuming sub-expressions.
  -->
                  
   <xsl:accumulator name="a" initial-value="0">
     <xsl:accumulator-rule match="*" select="$value + accumulator-before('a')"/>
   </xsl:accumulator>

   <xsl:template name="main">
      <out>
         <xsl:source-document streamable="true" href="error-0700a.xsl">
           <xsl:value-of select="accumulator-after('a')"/>
           <xsl:copy-of select="."/>
         </xsl:source-document>  
      </out>
   </xsl:template>   
</xsl:stylesheet>
