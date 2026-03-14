<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

                    <!--It is a static error if the component referenced by an xsl:override
                        declaration has visibility other than public or abstract-->
                        
   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-002" package-version="1.0.0">
  
     <xsl:override>
       <xsl:template name="t-private" as="xs:string" visibility="public">
         <xsl:sequence select="'London'"/>
       </xsl:template>
     </xsl:override>
      
   </xsl:use-package>   
   
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:stylesheet>
