<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                    <!--It is a dynamic error if the invocation of the stylesheet 
                        specifies a function name and arity that does not match the
                        expanded QName and arity of a named 
                        stylesheet function defined in the stylesheet,
                        whose visibility is public or final.-->
                        
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
   
</xsl:stylesheet>
