<?xml version="1.0" encoding="UTF-8"?>
<xsl:package version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"  declared-modes="no"
>
	<xsl:mode on-no-match="shallow-skip" use-accumulators="word-count"/>

    <!-- Keep a Word Count
    	Suppose that there is a requirement to output, at the end of the HTML rendition of a document, a paragraph giving the total number of words in the document. 
    	An accumulator can be used to maintain the word count: -->
	<xsl:accumulator name="word-count" 
                     as="xs:integer" 
                     initial-value="0">
    	<xsl:accumulator-rule match="text()" 
        	select="$value + count(tokenize(string(.), '\W+'))"/>
  	</xsl:accumulator>

	<!-- The final value can be output at the end of the document: -->
	<xsl:template match="/">
    	<xsl:apply-templates/>
		<p>Word count: <xsl:value-of select="accumulator-after('word-count')"/></p>
	</xsl:template>

</xsl:package>
