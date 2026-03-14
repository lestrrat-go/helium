<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
                 <!--It is a static error if an
                     xsl:merge-key element with a select attribute
                     has non-empty content.-->
   <xsl:template name="main">
      <out>
         <xsl:merge>
           <xsl:merge-source name="a" select="1 to 10">
             <xsl:merge-key select="."/>
           </xsl:merge-source>
           <xsl:merge-source name="b" select="2 to 10">
             <xsl:merge-key select=".">
               <xsl:sequence select=".+1"/>
             </xsl:merge-key>
           </xsl:merge-source>
           <xsl:merge-action>
             22
           </xsl:merge-action>
         </xsl:merge>
      </out>
   </xsl:template>
</xsl:stylesheet>
