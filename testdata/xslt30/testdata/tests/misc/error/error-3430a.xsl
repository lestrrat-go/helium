<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!--
      Error: It is a static error if a package contains a construct that is declared to be 
      streamable but which is not guaranteed-streamable, unless the user has indicated that the processor 
      is to handle this situation by processing the stylesheet without streaming or by making use of 
      processor extensions to the streamability rules where available.
-->
   
   <xsl:template name="main">
      <out>
        <xsl:source-document streamable="true" href="error-3430a.xsl">
          <xsl:value-of select="//a + //b"/>
        </xsl:source-document>         
      </out>
   </xsl:template>
</xsl:stylesheet>
