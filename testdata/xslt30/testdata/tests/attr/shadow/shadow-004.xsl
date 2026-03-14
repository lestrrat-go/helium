<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" _version="{'3.0'}" version="1.0">

   <!-- shadow attribute (static AVT) on xsl:include/@href -->
   <!-- A 3.0 processor does NOT use backwards-compatible mode here -->
   
   <xsl:variable name="V" static="yes" select="'003'"/>
   <xsl:include _href="shadow-{$V}.xsl"/>
   
   <xsl:template name="go">
      <xsl:call-template name="main"/>
   </xsl:template>
   
</xsl:transform>
