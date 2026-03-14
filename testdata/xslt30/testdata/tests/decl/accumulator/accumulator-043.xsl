<?xml version="1.0" encoding="UTF-8"?>
<xsl:package version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    exclude-result-prefixes="xs map"  declared-modes="no"
>
	<!-- Compute a Histogram showing the Number of Books, by Publisher
         The contained sequence constructor is evaluated with the variable $value set to the current value, and with the context node as the node being visited. -->
	<xsl:accumulator name="histogram" as="map(xs:untypedAtomic, xs:integer)"
    	initial-value="map{}">
	    <xsl:accumulator-rule match="ITEM">
    	  	<xsl:choose>
        		<xsl:when test="map:contains($value, PUBLISHER)">
	          		<xsl:sequence select="map:put($value, PUBLISHER, $value(PUBLISHER)+1)"/>
        		</xsl:when>
        		<xsl:otherwise>
          			<xsl:sequence select="map:put($value, PUBLISHER, 1)"/>
        		</xsl:otherwise>
      		</xsl:choose>
    	</xsl:accumulator-rule>  
	</xsl:accumulator>

	<xsl:template match="/">
		<histogram>
			<xsl:variable name="map" select="accumulator-after('histogram')"/>
			<xsl:for-each select="map:keys($map)">
			    <xsl:sort select="."/>
				<publisher name="{.}">
					<xsl:value-of select="map:get($map, .)"/>
				</publisher>
			</xsl:for-each>
        </histogram>  		
	</xsl:template>
	
	<xsl:mode use-accumulators="histogram"/>

</xsl:package>
