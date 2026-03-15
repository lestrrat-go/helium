<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
                    <!--It is a type error if the result of
                        evaluating the select expression of the
                        xsl:copy element is a sequence of more
                        than one item.-->
  
   <xsl:variable name="v" as="element()*">
     <x/>
     <y/>
   </xsl:variable>
     
   <xsl:template name="main">
      <out>
         <xsl:copy select="$v"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
