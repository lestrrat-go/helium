<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

              <!--It is a dynamic error if the
                  current-merge-group function is used when the current
                  merge group is absent. The error
                  may be reported statically if it can be detected
                  statically.-->
                  
                  
   <xsl:template name="main">
      <out>
         <xsl:copy-of select="current-merge-group()"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
