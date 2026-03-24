<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                    <!--It is a static error if the xsl:use-package elements in a
                        package manifest cause two or more homonymous 
                        components to be accepted with a visibility other than hidden.-->

   
   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-002" package-version="1.0.0">
     <xsl:accept component="template" names="*" visibility="public"/>
   </xsl:use-package> 
   
   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-003" package-version="1.0.0">
     <xsl:accept component="template" names="*" visibility="public"/>
   </xsl:use-package>                   
                        
                        
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:stylesheet>
