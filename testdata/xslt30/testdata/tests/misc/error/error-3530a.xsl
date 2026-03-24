<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                 <!--It is a dynamic error
                     if an xsl:try instruction is unable to recover the state of a final result tree
                     because recovery has been disabled by use of the attribute rollback-output="no".-->
                     
   <xsl:param name="p" select="0"/>
   
   <xsl:template name="main">
      <out>
         <xsl:try rollback-output="no">
           <xsl:for-each select="1 to 5000">
             <a><xsl:value-of select="."/></a>
           </xsl:for-each>
           <b><xsl:value-of select="2000 div $p"/></b>
           <xsl:catch errors="*">
             <c>caught</c>
           </xsl:catch>
         </xsl:try>
      </out>
   </xsl:template>
</xsl:stylesheet>
