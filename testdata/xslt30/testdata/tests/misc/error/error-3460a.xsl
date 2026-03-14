<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  name="http://www.w3.org/xslt30tests/error-3460a"
  declared-modes='no'>

<!--
      It is a static error if an xsl:apply-imports element appears in a template rule 
      declared within an xsl:override element. (To invoke the template rule that is being overridden, 
      xsl:next-match should therefore be used.)
-->
   
   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-005" package-version="1.0.0">
     <xsl:override>
     
       <xsl:template match="a" mode="m">
         <a><xsl:apply-imports/></a>
       </xsl:template>
       
     </xsl:override>
   </xsl:use-package>
   
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:package>
