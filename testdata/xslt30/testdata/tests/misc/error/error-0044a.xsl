<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                    <!--It is a dynamic error if the invocation of the stylesheet 
                        specifies an initial mode and if no initial match selection 
                        is supplied (either explicitly, or
                        defaulted to the global context item).
                    -->
                    
   <xsl:template match="/" mode="aMode">
      <out>
         <x/>
      </out>
   </xsl:template>
   
</xsl:stylesheet>
