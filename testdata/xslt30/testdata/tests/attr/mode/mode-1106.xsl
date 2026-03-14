<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   >
<!-- Purpose: Use of accumulators that are / are not applicable to the initial mode. -->

   <xsl:output method="xml" encoding="UTF-8" indent="no"/>
   
   <xsl:param name="STREAMABLE" select="false()" static="true"/>
   
   <xsl:accumulator name="counter" initial-value="0" as="xs:integer" _streamable="{$STREAMABLE}">
      <xsl:accumulator-rule match="*" select="$value + 1"/>
   </xsl:accumulator>
   
   <xsl:accumulator name="counter2" initial-value="0" as="xs:integer" _streamable="{$STREAMABLE}">
      <xsl:accumulator-rule match="*" select="$value + 1"/>
   </xsl:accumulator>
   
   <xsl:mode name="Q" _streamable="{$STREAMABLE}"/>
   
   <xsl:mode name="X" use-accumulators="counter" _streamable="{$STREAMABLE}"/>
   
   <xsl:mode name="Y" use-accumulators="" _streamable="{$STREAMABLE}"/>
   
   <xsl:mode name="Z" use-accumulators="#all" _streamable="{$STREAMABLE}"/>
   
   <xsl:mode use-accumulators="counter" _streamable="{$STREAMABLE}"/>

   <xsl:template match="/" mode="X&#x9;Y&#xa;Z&#xd;#unnamed">
	  <out>
	 	 <xsl:value-of select="accumulator-after('counter')"/>
      </out>
   </xsl:template>


</xsl:transform>
