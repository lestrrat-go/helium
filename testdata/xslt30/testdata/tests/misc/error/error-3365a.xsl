<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

              <!--A dynamic error 
                  occurs if the set of keys in the maps resulting from 
                  evaluating the sequence constructor within an xsl:map 
                  instruction contains duplicates.-->
                  
   <xsl:template name="main">
      <out>
         <xsl:variable name="v" as="map(*)">
         <xsl:map>
           <xsl:for-each select="1 to 20">
              <xsl:map-entry key=". mod 10" select="true()"/>
           </xsl:for-each>
         </xsl:map>
         </xsl:variable>
         <xsl:value-of select="count($v)"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
