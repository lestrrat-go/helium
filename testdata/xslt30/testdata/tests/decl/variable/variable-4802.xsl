<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
   expand-text="yes">
   
   <xsl:output method="xml" indent="no" encoding="UTF-8"/>
   
   <xsl:mode on-no-match="shallow-skip"/>
   
   <xsl:param name="servletName" select="'MyServlet'"/>
   <xsl:param name="urlPattern" select='concat("/servlet/",$servletName,"/*")'/>  <!-- Default to normal servlet URL pattern -->
   
   <xsl:template match="servlet-mapping[servlet-name=$servletName] / url-pattern[position()=last()]">
      <out>{.}</out>
   </xsl:template>
   
   <xsl:template match="url-pattern"><wrong/></xsl:template>
   

</xsl:stylesheet>