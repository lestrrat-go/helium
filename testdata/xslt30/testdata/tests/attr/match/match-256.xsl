<xsl:stylesheet version="2.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    
    <xsl:variable name="atomic" select="93"/>
    
    <!-- A match pattern in the form of a variable reference will only match nodes -->
    
    <!-- Analysis (MHK 2017-09-15). The semantics of this pattern are that it will match an item N if:
        
        N is a node, and the result of evaluating the expression root(.)//($atomic) with a singleton focus 
        based on N is a sequence that includes the node N.
        
        Now, if N is a node, the result of this expression will be a sequence consisting of multiple occurrences
        of the integer 93. This sequence does not include N, therefore the pattern does not match. There is no
        dynamic error, and no type error, so no rule in the specification is violated and the most the processor
        can do is to issue a warning.
    -->
    
    <xsl:template match="$atomic">
        <wrong/>
    </xsl:template>
    
    <xsl:mode on-no-match="deep-skip"/>
    
    <xsl:template name="xsl:initial-template">
        <out>
            <xsl:apply-templates select="93" />
        </out>
    </xsl:template>
    
</xsl:stylesheet>
