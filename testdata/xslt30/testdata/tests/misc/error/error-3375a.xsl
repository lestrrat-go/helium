<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

              <!--A type error  occurs if the result of 
                  evaluating the sequence constructor within an xsl:map instruction is not an instance
                  of the required type map(*)*.
              -->
              
   <xsl:template name="main">
      <out>
         <xsl:variable name="v" as="map(*)">
           <xsl:map>
              <xsl:for-each select="1 to 10">
                <xsl:if test=". lt 5">
                  <xsl:map-entry key="." select="."/>
                </xsl:if>
                <xsl:if test=". ge 5">
                  <xsl:sequence select="."/>
                </xsl:if>
              </xsl:for-each>
           </xsl:map>
         </xsl:variable>
         <xsl:value-of select="count($v)"/>  
      </out>
   </xsl:template>
</xsl:stylesheet>
