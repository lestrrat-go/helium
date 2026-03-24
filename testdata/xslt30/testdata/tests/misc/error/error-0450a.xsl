<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                             <!--It is a non-recoverable dynamic
                                 error if the result sequence contains a function item.
                             -->
                             
   <xsl:template name="main">
      <out>
         <x>
           <xsl:sequence select="true#0"/>
         </x>
      </out>
   </xsl:template>
</xsl:stylesheet>
