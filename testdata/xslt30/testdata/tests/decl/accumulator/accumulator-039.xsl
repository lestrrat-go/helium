<xsl:package version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    exclude-result-prefixes="xs"  declared-modes="no">
    
    <!-- Use local and global variables in xsl:accumulator/@initial-value 
         and @new-value -->
    
    <xsl:param name="g" select="1"/>
    
    <xsl:variable name="v">
      <xsl:for-each select="1 to 50">
         <w/>
      </xsl:for-each>
    </xsl:variable>
    
    <xsl:template match="/" name="main" visibility="public">
        <out>
          <last><xsl:value-of select="$v/w[last()]/accumulator-after('big')"/></last>
          <first><xsl:value-of select="$v/w[1]/accumulator-after('big')"/></first>
        </out>
    </xsl:template>
    
    <xsl:accumulator name="big" initial-value="for $i in $g to 100 return $i*$i" 
               streamable="no">
        <xsl:accumulator-rule match="*" 
                              select="for $i in 1 to $g return remove($value, $i)" phase="start"/>
        
    </xsl:accumulator>

    
</xsl:package>