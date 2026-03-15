<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
<!-- Purpose: xsl:element with on-empty -->

  <t:param name="n">name</t:param>
  <t:param name="jaq" as="element()"><jaq/></t:param>

  <t:template match="/">
	<out>
	  <one>
	    <t:where-populated>
          <t:element name="{$n}">
            <t:where-populated>
            <t:document>
			  <t:if test="current-date() gt xs:date('2000-01-01')"><e/></t:if>
			</t:document>  
			</t:where-populated>
		  </t:element>
		 </t:where-populated>
		 <t:on-empty select="$jaq"/> 
	  </one>
	  <two>
		 <t:where-populated>
          <t:element name="{$n}">
            <t:where-populated>
            <t:document>
			  <t:if test="current-date() lt xs:date('2000-01-01')"><e/></t:if>
			</t:document>  
			</t:where-populated>
		  </t:element>
		 </t:where-populated>
		 <t:on-empty select="$jaq"/> 
	  </two>	  
    </out>
  </t:template>
  
</t:transform>
