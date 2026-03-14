<?xml version="1.0" encoding="UTF-8"?>
<xsl:package name="http://www.w3.org/package/error-3020a" 
            xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
            
                    <!--It is a static error if an xsl:expose element matches no
                        components in the containing package, unless the tokens in the names
                        attribute are all wildcards.-->
                        
                    <!-- See also bug 26679 -->    
                        
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
   
   <xsl:expose component="template" names="p" visibility="public"/>
   
</xsl:package>
