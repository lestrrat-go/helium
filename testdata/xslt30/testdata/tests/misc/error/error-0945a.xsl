<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
                xmlns:f="http://example.com/f">

                    <!--It is a type error
                        to use the xsl:copy instruction with no select attribute when the context item is absent.
                     -->
                     
   <xsl:template name="main">
      <out>
         <x y="{f:f()}"/>
      </out>
   </xsl:template>
   
   <xsl:function name="f:f">
     <xsl:copy>
       <a/>
     </xsl:copy>
   </xsl:function>
   
</xsl:stylesheet>
