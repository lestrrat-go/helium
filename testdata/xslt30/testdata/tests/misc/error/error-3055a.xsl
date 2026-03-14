<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
>

                    <!--It is a static error 
                        if a component declaration appearing as a child of xsl:override
                        is homonymous with
                        any other declaration in the using package, regardless of 
                        import precedence, including any other
                        overriding declaration in the package manifest of the using package.-->

   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-002" package-version="1.0.0">
  
     <xsl:override>
       <xsl:template name="t-public" as="xs:string" visibility="public">
         <xsl:sequence select="'London'"/>
       </xsl:template>
     </xsl:override>
      
   </xsl:use-package> 
   
   <xsl:template name="t-public" as="xs:string" visibility="public">
         <xsl:sequence select="'Paris'"/>
   </xsl:template>
                        
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:stylesheet>
