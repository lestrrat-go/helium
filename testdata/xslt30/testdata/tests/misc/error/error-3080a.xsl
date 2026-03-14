<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  name="http://www.w3.org/xslt30tests/error-3080" package-version="1.0">

                    <!--It is a static error 
                        if a top-level package intended for execution (as distinct from a library
                        package) contains symbolic references
                        referring to components whose visibility is abstract.-->
                        
   <xsl:template name="main">
      <out>
         <xsl:call-template name="t"/>
      </out>
   </xsl:template>
   
   <xsl:template name="t" visibility="abstract"/>
   
</xsl:package>
