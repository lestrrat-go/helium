<?xml version="1.0" encoding="UTF-8"?>
<xsl:package version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"  declared-modes="no"
>
	<xsl:output omit-xml-declaration="yes" />
	<xsl:strip-space elements="*"/>
	<xsl:mode use-accumulators="section-nr"/>

	<!-- Output Hierarchic Section Numbers
         Consider a document in which section elements are nested within section elements to arbitrary depth,
         and there is a requirement to render the document with hierarchic section numbers of the form 3.5.1.4.

		 The current section number can be maintained in an accumulator in the form of a sequence of integers, managed as a stack.
		 The number of integers represents the current level of nesting, and the value of each integer represents the number of preceding sibling sections encountered at that level.
		 For convenience the first item in the sequence represents the top of the stack. -->

	<xsl:accumulator name="section-nr" as="xs:integer*" 
                 	 initial-value="0">
		<xsl:accumulator-rule match="section" phase="start" 
                 	 select="0, head($value)+1, tail($value)"/>
		<xsl:accumulator-rule match="section" phase="end" 
                 	 select="tail($value) (:pop:)"/>
	</xsl:accumulator>
	
	<xsl:template match="/">
		<out><xsl:apply-templates/></out>
	</xsl:template>

	<!-- The section number for a section can thus be generated as: -->	
	<xsl:template match="section">
		<p>
			<xsl:value-of select="reverse(tail(accumulator-before('section-nr')))" 
                  		  separator="."/>
        </p>
  		<xsl:apply-templates/>
	</xsl:template>

</xsl:package>
