<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
              <!--If the result of evaluating the 
                  context-item expression of an xsl:evaluate instruction
                  is a sequence containing more than one item, then a type error
                   is signaled.-->
   <xsl:template name="main">
      <out>
         <xsl:variable name="v" as="element()*">
           <x/><y/>
         </xsl:variable>
         <xsl:evaluate xpath="concat('//', 'x')" context-item="$v"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
