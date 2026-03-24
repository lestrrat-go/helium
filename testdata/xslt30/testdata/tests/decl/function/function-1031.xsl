<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:x="http://xxx.com/" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    version="3.0"
    exclude-result-prefixes="xs x">

    <!-- 
        cached function. Although implementations are not required to respect the request
        for caching, this function will be tediously slow without it        
    -->
    <?spec xslt#stylesheet-functions?>

    <xsl:function name="x:fib" cache="yes" as="xs:integer">
        <xsl:param name="n" as="xs:integer" />
        <xsl:sequence
            select="
                if ($n = (1, 2)) then
                    1
                else
                    x:fib($n - 1) + x:fib($n - 2)"/>
    </xsl:function>

    <xsl:template match="/">
        <out>
            <xsl:value-of select="x:fib(92)" />
        </out>
    </xsl:template>


</xsl:stylesheet>
