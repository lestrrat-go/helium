<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: test base URI of nodes in result document. -->

   <xsl:output method="xml" encoding="UTF-8"/>

   <xsl:template name="main">
      <out>
         <log>Before redirect</log>
         <xsl:result-document href="out/second.xml">
            <two>
               <k>Kilo</k>
               <l xml:base="in/third.xml">Lima</l>
            </two>
         </xsl:result-document>
         <log>After redirect</log>
      </out>
   </xsl:template>
</xsl:transform>
