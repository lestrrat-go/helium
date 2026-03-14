<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:f="http://f.com/"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="#all" expand-text="true">
   <!-- Purpose: Attempt to sort a sequence containing untyped attribute nodes and dates.-->
   <!-- Should fail because "lt" comparison between xs:untypedAtomic and xs:date is not allowed. -->
   
   <xsl:variable name="in">
      <e at="2017-01-05"/>
      <e at="2014-01-02"/>
      <e at="2016-03-02"/>     
   </xsl:variable>

   <xsl:template name="xsl:initial-template">
      <out>
         <xsl:for-each select="$in/e/@at, xs:date('2011-12-31')">
            <xsl:sort select="."/>
            <item>{.}</item>
         </xsl:for-each>
      </out>
   </xsl:template>

</xsl:transform>
