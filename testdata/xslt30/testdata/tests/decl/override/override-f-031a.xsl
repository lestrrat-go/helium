<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://localhost/fn"
    name="http://london2016/override-f-030/package-1"
    package-version="0.0.1"
    exclude-result-prefixes="xs f"
    version="3.0">
    
    <!-- identity of union types -->
    
    <xsl:import-schema>
        <xs:schema>
            <xs:simpleType name="u1">
                <xs:union  memberTypes="xs:decimal xs:double"/>
            </xs:simpleType>
        </xs:schema>
    </xsl:import-schema>
    
    <xsl:expose component="function" names="*" visibility="public"/>
    
    <xsl:function name="f:a" visibility="public" as="u1">
        <xsl:param name="x" as="u1"/>
        <xsl:sequence select="$x"/>
    </xsl:function> 
    
    <xsl:function name="f:b" visibility="public" as="u1">
        <xsl:param name="x" as="u1"/>
        <xsl:sequence select="$x"/>
    </xsl:function> 
    
    
    
</xsl:package>