<xsl:stylesheet version='3.0' xmlns:xsl='http://www.w3.org/1999/XSL/Transform'>
    
    <xsl:mode streamable='yes' on-no-match='shallow-copy'/>
    
    <xsl:import-schema schema-location="../docs/books.xsd"/>
    
    <xsl:template match='schema-element(ITEM)/AUTHOR'/> 
</xsl:stylesheet>