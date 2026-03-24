<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                    <!--It is a static error if the number of
                        xsl:merge-key children of a
                        xsl:merge-source element is not equal to the number of
                        xsl:merge-key children of another
                        xsl:merge-source child of the same
                        xsl:merge instruction.
                    -->
                    
   <xsl:template name="main">
      <out>
         <xsl:merge>
           <xsl:merge-source name="a" select="1 to 50">
             <xsl:merge-key select="."/>
           </xsl:merge-source>
           <xsl:merge-source name="b" select="1 to 50">
             <xsl:merge-key select=". div 10"/>
             <xsl:merge-key select=". mod 10"/>
           </xsl:merge-source>
           <xsl:merge-action>
             22
           </xsl:merge-action>
         </xsl:merge>
      </out>
   </xsl:template>
</xsl:stylesheet>
