<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:f="http://example.com/f">
              <!--The two attributes for-each-item and
               for-each-source are mutually exclusive. 
               The xsl:merge-source element may contain
               either or neither, but not both. It is a static error if both are present.-->
   <xsl:template name="main">
      <out>
         <xsl:merge>
           <xsl:merge-source
             for-each-item="1 to 10"
             for-each-source="'a.xml', 'b.xml'"
             select=".">
             <xsl:merge-key select="."/>
           </xsl:merge-source>
           <xsl:merge-action/>
         </xsl:merge>            
      </out>
   </xsl:template>
   
</xsl:stylesheet>
