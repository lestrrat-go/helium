<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

              <!--It is a dynamic error if the
                  $source argument of the current-merge-group
                  function does not match the name attribute of any
                  xsl:merge-source element for the current merge operation. The
                  error may be reported statically if it can be detected
                  statically.-->
                  
   <xsl:template name="main">
      <out>
         <xsl:merge>
           <xsl:merge-source name="a" select="1 to 30">
             <xsl:merge-key select="."/>
           </xsl:merge-source>
           <xsl:merge-source name="b" select="1 to 30">
             <xsl:merge-key select="."/>
           </xsl:merge-source>
           <xsl:merge-action>
             <xsl:copy-of select="current-merge-group('c')"/>
           </xsl:merge-action>
         </xsl:merge>
      </out>
   </xsl:template>
</xsl:stylesheet>
