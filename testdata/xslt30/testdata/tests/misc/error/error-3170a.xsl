<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
                             <!--It is a type error if the
                                 result of evaluating the namespace-context attribute
                                 of the xsl:evaluate instruction is anything other
                                 than a single node.-->
   
   <xsl:variable name="v" as="element()*">
     <x/>
     <y/>
   </xsl:variable>
   
   <xsl:template name="main">
      <out>
         <xsl:evaluate xpath="'1 to 10'" namespace-context="$v"/> 
      </out>
   </xsl:template>
</xsl:stylesheet>
